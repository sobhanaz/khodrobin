# Build context is the repo root so the seed snapshot can be baked in.
FROM python:3.12-slim
WORKDIR /app
COPY services/crawler/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY services/crawler/ ./
# The seed guarantees a non-empty index on first boot, before any crawl has run.
COPY data/seed/ /seed/
ENTRYPOINT ["python", "-u", "schedule.py"]
