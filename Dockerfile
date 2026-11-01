# Build
FROM golang:alpine AS build
WORKDIR /build/
COPY go.mod .
RUN go mod tidy
COPY . .
RUN go build

# Production
FROM alpine AS production
WORKDIR /monitordatamanager/
COPY --from=build /build/monitordatamanager ./
COPY ./docs ./docs
COPY ./*.crt ./
COPY ./*.key ./
COPY ./monitordatamanager.json ./
COPY ./*.tmpl ./

EXPOSE 6006 10051

CMD ["./monitordatamanager", "-log=stdout"]