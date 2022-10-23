# Start the Go app build
FROM golang:latest AS build

# Copy source
WORKDIR /project/source
COPY . .

# Get required modules
RUN go mod tidy

# Build a statically-linked Go binary for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -a -o main .

# New build phase -- create binary-only image
FROM alpine:latest

# Add support for HTTPS
RUN apk update && \
    apk upgrade && \
    apk add ca-certificates

WORKDIR /project

# Copy files from previous build container
COPY --from=build /project/source/main ./

# Add environment variables
# ENV ...
ENV Loggly_Token http://logs-01.loggly.com/inputs/5e085983-7ed1-4fc1-bf95-5f6278278035/tag/http/

# Check results
RUN env && pwd && find .

# Start the application
CMD ["./main"]

# Remember start the image with
# docker run -p <local port>:<inside port> -d <image name>