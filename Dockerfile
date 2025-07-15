FROM python:alpine3.18 AS base

WORKDIR /app

FROM base AS setup
COPY /app/requirements.txt requirements.txt
RUN pip install --upgrade pip setuptools
RUN pip install --no-cache-dir --upgrade -r requirements.txt

FROM setup AS main
COPY app/main.py main.py
COPY app/test_main.py test_main.py
COPY app/__init__.py __init__.py

EXPOSE 8080
ENTRYPOINT ["fastapi", "run", "/app/main.py", "--port", "8080"]
