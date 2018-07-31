FROM jrottenberg/ffmpeg
WORKDIR /
RUN mkdir /music && mkdir /cred
ADD blindbot /
ADD player.html /
CMD ["/blindbot"]
ENTRYPOINT  [""]
