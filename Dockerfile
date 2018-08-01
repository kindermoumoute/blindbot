FROM jrottenberg/ffmpeg
WORKDIR /

ADD blindbot /
ADD player.html /
RUN mkdir /music /cred /db

CMD ["/blindbot"]
ENTRYPOINT  [""]
