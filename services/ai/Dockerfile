FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY app/ ./app/
COPY evals/ ./evals/
EXPOSE 8000
# One worker: the cache is in-process, and a second worker would halve its hit
# rate while doubling model load for no throughput we need at this traffic.
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "1"]
