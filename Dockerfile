FROM jrottenberg/ffmpeg
WORKDIR /
RUN mkdir /music
ADD blindbot /
ADD player.html /
CMD ["/blindbot"]
ENTRYPOINT  [""]
