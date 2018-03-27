FROM scratch

WORKDIR /
COPY ./discord-lolstatus /

VOLUME ["/data"]

ENTRYPOINT ["./discord-lolstatus"]
