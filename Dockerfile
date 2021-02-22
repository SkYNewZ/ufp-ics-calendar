FROM golang:1.15-alpine3.12
WORKDIR /go/src/github.com/SkYNewZ/ufp-ics-calendar
COPY go.* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /ufp-ics-calendar .

FROM scratch

ARG BUILD_DATE
ARG VCS_REF

LABEL maintainer="Quentin Lemaire <quentin@lemairepro.fr>"
LABEL org.label-schema.schema-version="1.0"
LABEL org.label-schema.build-date=$BUILD_DATE
LABEL org.label-schema.name="ufp-ics-calendar"
LABEL org.label-schema.description="Generate iCal from UFP web portal"
LABEL org.label-schema.url="https://github.com/SkYNewZ/ufp-ics-calendar"
LABEL org.label-schema.vcs-ref=$VCS_REF

COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /ufp-ics-calendar /ufp-ics-calendar
ENTRYPOINT ["/ufp-ics-calendar"]

