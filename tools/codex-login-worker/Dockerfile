FROM python:3.11-slim
RUN apt-get update && apt-get install -y --no-install-recommends nodejs ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /worker
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
RUN useradd --uid 10001 --create-home worker
USER worker
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
EXPOSE 8081
CMD ["python", "server.py"]
