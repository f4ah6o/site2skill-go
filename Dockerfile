FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    wget \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install uv
RUN pip install uv

# Set working directory
WORKDIR /workspace

# Copy the entire project
COPY . /app/site2skill-go

# Test installation from local path
RUN uvx --from /app/site2skill-go site2skillgo --help

# Default command shows help
CMD ["uvx", "--from", "/app/site2skill-go", "site2skillgo", "--help"]
