FROM gcr.io/distroless/static

COPY autopgo /usr/bin/autopgo
COPY README.md /usr/bin/README.md

CMD ["/usr/bin/autopgo"]
