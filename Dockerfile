FROM gcr.io/distroless/static

COPY autopgo /usr/bin/autopgo
COPY README.md /usr/bin/README.md
COPY LICENSE /usr/bin/LICENSE

CMD ["/usr/bin/autopgo"]
