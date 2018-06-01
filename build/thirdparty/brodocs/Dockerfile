# Builds Docker image to run brodocs

FROM node:7.2
MAINTAINER Phillip Wittrock <pwittroc@google.com>

RUN apt-get update && apt-get install -y npm git && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN git clone --depth=1 https://github.com/Birdrock/brodocs.git
WORKDIR brodocs
RUN npm install

COPY runbrodocs.sh .

RUN rm -rf ./documents/*
RUN chmod -R 777 .

CMD ["./runbrodocs.sh"]
