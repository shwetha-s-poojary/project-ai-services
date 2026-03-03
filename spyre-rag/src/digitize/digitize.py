import json
import time
from pathlib import Path

import common.db_utils as db
from common.misc_utils import *
from digitize.status import StatusManager, DocStatus, JobStatus
from digitize.pdf_utils import get_pdf_page_count
from concurrent.futures import ProcessPoolExecutor

from docling.datamodel.document import DoclingDocument, TextItem

logger = get_logger("ingest")

def digitize(directory_path, job_id=None, doc_id_dict: dict = None, output_format: str = "json"):
    """
    Digitize a single PDF file in the staging directory.
    Converts to JSON and optionally Markdown or text, updates metadata, but does not return anything.
    
    Args:
        directory_path: Path to staging directory containing exactly one PDF
        job_id: Job identifier for StatusManager
        doc_id_dict: Mapping from filename to document ID
        output_format: "json", "md", or "text"
    """
    directory_path = Path(directory_path)
    if not directory_path.exists():
        raise Exception(f"Staging directory does not exist: {directory_path}")

    # Initialize StatusManager
    status_mgr = StatusManager(job_id) if job_id else None

    # Prepare output/cache path
    vector_store = db.get_vector_store()
    index_name = vector_store.index_name
    out_path = setup_cache_dir(index_name)

    # Ensure exactly one PDF file
    pdfs = list(directory_path.glob("*.pdf"))
    if len(pdfs) != 1:
        raise Exception(f"Expected exactly one PDF in {directory_path}, found {len(pdfs)}")

    file_path = pdfs[0]
    filename = file_path.name
    doc_id = doc_id_dict.get(filename)
    if doc_id is None:
        raise Exception(f"Document ID not found for {filename}")

    logger.info(f"Digitization started for '{filename}'")

    try:
        # Mark document/job as IN_PROGRESS
        if status_mgr:
            status_mgr.update_doc_metadata(doc_id, {
                "status": DocStatus.IN_PROGRESS,
                "started_at": status_mgr._get_timestamp()
            })
            status_mgr.update_job_progress(doc_id, DocStatus.IN_PROGRESS, JobStatus.IN_PROGRESS)
        logger.info("updating metadata to IN_PROGRESS")

        # Checksum handling
        logger.info("Checking if document conversion needed")
        checksum_path = Path(out_path) / f"{doc_id}.checksum"
        json_path = Path(out_path) / f"{doc_id}.json"
        convert_needed = True
        new_checksum = generate_file_checksum(file_path)

        if checksum_path.exists():
            cached_checksum = checksum_path.read_text().strip()
            if cached_checksum == new_checksum and json_path.exists():
                convert_needed = False

        if convert_needed:
            checksum_path.write_text(new_checksum, encoding="utf-8")
            logger.info(f"Conversion needed, converting document")
            # Convert document
            # Run conversion inside a single process worker
            with ProcessPoolExecutor(max_workers=1) as executor:
                future = executor.submit(
                    convert_document,
                    str(file_path),
                    {"convert": convert_needed},
                    Path(out_path)
                )
                _, converted_json_path, conversion_time = future.result()

            if not converted_json_path:
                raise Exception("Conversion failed")
        else:
            converted_json_path=json_path
            conversion_time = 0.0

        logger.info(f"Loading json data")
        # Load JSON → DoclingDocument
        try:
            with open(converted_json_path, "r", encoding="utf-8") as f:
                raw_data = json.load(f)
        except json.JSONDecodeError as e:
            logger.exception(f"Failed to load JSON for {filename}: {e}")
            raise

        doc_obj = DoclingDocument.model_validate(raw_data)

        # Save requested output format
        output_format = output_format.lower()

        if output_format not in {"json", "md", "text"}:
            raise ValueError(f"Unsupported output_format: {output_format}")

        if output_format == "md":
            logger.info(f"Saving in .md format")
            md_path = Path(out_path) / f"{doc_id}.md"
            with open(md_path, "w", encoding="utf-8") as f:
                f.write(doc_obj.export_to_markdown())

        elif output_format == "text":
            logger.info(f"Saving in .txt format")
            txt_path = Path(out_path) / f"{doc_id}.txt"
            with open(txt_path, "w", encoding="utf-8") as f:
                f.write(doc_obj.export_to_text())

        # Collect metadata
        page_count = get_pdf_page_count(str(file_path))

        # Mark COMPLETED
        logger.info("updating metadata to COMPLETED")
        if status_mgr:
            status_mgr.update_doc_metadata(doc_id, {
                "status": DocStatus.COMPLETED,
                "pages": page_count,
                "timing_in_secs": {"digitizing": round(conversion_time, 2)},
                "completed_at": status_mgr._get_timestamp()
            })
            status_mgr.update_job_progress(doc_id, DocStatus.COMPLETED, JobStatus.COMPLETED)

    except Exception as e:
        # Mark FAILED
        logger.info("updating metadata to FAILED")
        # TO-DO 
        # logger.exception(f"Digitization failed for {filename}: {e}")
        if status_mgr:
            status_mgr.update_doc_metadata(doc_id, {"status": DocStatus.FAILED})
            status_mgr.update_job_progress(doc_id, DocStatus.FAILED, JobStatus.FAILED)
        raise