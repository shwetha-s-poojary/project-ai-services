import asyncio
from datetime import datetime, timezone
from enum import Enum
from functools import partial
import json
from pathlib import Path
from typing import List, Optional
import uuid
from common.misc_utils import get_logger

CACHE_DIR = "/var/cache"
DOCS_DIR = f"{CACHE_DIR}/docs"
JOBS_DIR = f"{CACHE_DIR}/jobs"

logger = get_logger("digitize_utils")

class OutputFormat(str, Enum):
    TEXT = "text"
    MD = "md"
    JSON = "json"

class OperationType(str, Enum):
    INGESTION = "ingestion"
    DIGITIZATION = "digitization"

class JobStatus(str, Enum):
    ACCEPTED = "accepted"
    IN_PROGRESS = "in_progress"
    COMPLETED = "completed"
    FAILED = "failed"

class DocStatus(str, Enum):
    ACCEPTED = "accepted"
    IN_PROGRESS = "in_progress"
    COMPLETED = "completed"
    FAILED = "failed"

def generate_job_id():
    # Generate a random UUID
    job_id = uuid.uuid4()
    job_id_hex = job_id.hex
    print(f"Hex-only ID: {job_id_hex}")
    print(f"job id : {job_id}")
    return str(job_id)


def generate_document_id(filename):
    """
    Generate UUID based document_id based on filename, helps preventing duplicate document records 
    """
    # Define a fixed Namespace: use any valid UUID
    NAMESPACE_INGESTION = uuid.UUID('6ba7b810-9dad-11d1-80b4-00c04fd430c8')

    # Generate deterministic UUID
    document_id = uuid.uuid5(NAMESPACE_INGESTION, filename)
    return str(document_id)


def initialize_job_state(job_id: str, operation: str, documents_info: list, output_format: str):
    """
    Creates the job status file and individual document metadata files.
    documents_info: List of dicts with {'id': uuid, 'name': filename, 'type': op_type}
    """
    # Create docs and jobs dirs if not present already
    Path(DOCS_DIR).mkdir(parents=True, exist_ok=True)
    Path(JOBS_DIR).mkdir(parents=True, exist_ok=True)

    submitted_at = datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")

    # list to store documents in Job file
    job_documents_summary = []

    # dictionary to keep mapping of filename to document id.
    # key -> filename
    # val -> doc_id
    doc_id_dict = {}

    for doc in documents_info:
        # Create unique document id and document metadata for each files before spawning backgroundtask
        doc_id = generate_document_id(doc)
        doc_id_dict[doc] = doc_id
        logger.debug(f"Generated document id {doc_id} for the file: {doc}")

        # Create the document level metadata files (<doc_id>_metadata.json)
        doc_meta_path = Path(DOCS_DIR)/f"{doc_id}_metadata.json"
        doc_initial_data = {
            "id": doc_id,
            "name": doc,
            "type": operation,
            "status": DocStatus.ACCEPTED,
            "output_format": output_format,
            "completed_at": None,
            "error": "",
            "pages": 0,
            "tables": 0,
            "chunks": 0,
            "timing_in_secs": {
                "digitizing": None, "processing": None, "chunking": None, "indexing": None
            }
        }
        with open(doc_meta_path, "w") as f:
            json.dump(doc_initial_data, f, indent=4)

        logger.debug(f"Created document metadata file: {doc_meta_path}")

        # Add doc's summary to list for the Job file
        job_documents_summary.append({
            "id": doc_id,
            "name": doc,
            "status": DocStatus.ACCEPTED
        })

    # Create job status file (<job_id>_status.json)
    job_status_path = Path(JOBS_DIR) / f"{job_id}_status.json"

    job_data = {
        "job_id": job_id,
        "operation": operation,
        "status": JobStatus.ACCEPTED,
        "submitted_at": submitted_at,
        "last_updated_at": submitted_at,
        "documents": job_documents_summary,
        "error": ""
    }

    with open(job_status_path, "w") as f:
        json.dump(job_data, f, indent=4)

    logger.debug(f"Created job status file: {job_status_path}")

    return doc_id_dict


async def stage_upload_files(job_id: str, files: List[dict], staging_dir: str, file_contents: List[bytes]):
    base_stage_path = Path(staging_dir)
    base_stage_path.mkdir(parents=True, exist_ok=True)

    def save_sync(file_path: Path, content: bytes):
        with open(file_path, "wb") as f:
            f.write(content)
        return str(file_path)

    loop = asyncio.get_running_loop()

    for filename, content in zip(files, file_contents):
        target_path = base_stage_path / filename

        try:
            await loop.run_in_executor(
                None, 
                partial(save_sync, target_path, content)
            )
            print(f"Successfully staged file: {filename}")

        except Exception as e:
            logger.error(f"Failed to stage {filename} for job {job_id}: {e}")
            raise


def read_job_file(job_file: Path) -> Optional[dict]:
    """Reads and parses a single job status JSON file. Returns None on failure."""
    try:
        with open(job_file, "r") as f:
            return json.load(f)
    except (json.JSONDecodeError, IOError) as e:
        logger.warning(f"Skipping unreadable job file {job_file.name}: {e}")
        return None


def format_job_response(job_data: dict) -> dict:
    """Projects a raw job status dict down to the public API response shape.

    The on-disk format may carry internal fields (e.g. last_updated_at).
    This function returns only the fields defined in the design doc.
    """
    documents = job_data.get("documents", [])
    formatted_docs = [
        {
            "id": doc.get("id", ""),
            "name": doc.get("name", ""),
            "status": doc.get("status", ""),
        }
        for doc in documents
    ]

    return {
        "job_id": job_data.get("job_id", ""),
        "operation": job_data.get("operation", ""),
        "status": job_data.get("status", ""),
        "submitted_at": job_data.get("submitted_at", ""),
        "documents": formatted_docs,
        "error": job_data.get("error", ""),
    }


def load_all_jobs() -> List[dict]:
    """Loads every *_status.json from the jobs directory, sorted newest-first."""
    jobs_dir = Path(JOBS_DIR)
    if not jobs_dir.exists():
        return []

    all_jobs = []
    for job_file in jobs_dir.glob("*_status.json"):
        job_data = read_job_file(job_file)
        if job_data is not None:
            all_jobs.append(job_data)

    # Sort by submitted_at descending so the latest job is first
    all_jobs.sort(key=lambda j: j.get("submitted_at", ""), reverse=True)
    return all_jobs
