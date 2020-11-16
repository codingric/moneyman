FROM python:3.8.6-alpine3.12
MAINTAINER Ricardo Salinas <ricardo@salinas.id.au>

COPY requirements.txt /tmp/requirements.txt
RUN pip3 install --upgrade pip \
    && pip3 install -r /tmp/requirements.txt

ENV APP_DIR /app

RUN mkdir -p ${APP_DIR}/templates ${APP_DIR}/static
COPY templates ${APP_DIR}/templates
COPY *.py ${APP_DIR}

#COPY static ${APP_DIR}

#VOLUME ${APP_DIR}
EXPOSE 5000

# Cleanup
RUN rm -rf /.wh /root/.cache /var/cache /tmp/requirements.txt

WORKDIR ${APP_DIR}
CMD gunicorn --bind 0.0.0.0:$PORT rest-api:app
