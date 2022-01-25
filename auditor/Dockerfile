FROM frolvlad/alpine-python3
LABEL org.opencontainers.image.source https://github.com/codingric/moneyman

RUN pip install requests pyaml

CMD ["python3", "main.py"]

COPY main.py .

