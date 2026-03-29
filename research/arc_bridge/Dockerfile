FROM python:3.13-slim AS base

WORKDIR /app

# Install uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Copy project files
COPY pyproject.toml .
COPY *.py .

# Install dependencies
RUN uv sync --no-dev

# Run
EXPOSE 8787
CMD ["uv", "run", "uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8787"]
