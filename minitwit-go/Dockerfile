FROM golang:bullseye

# Set the working directory
WORKDIR /app

RUN go mod init minitwit

RUN go install github.com/cosmtrek/air@latest

#RUN go mod download
# Copy the server code into the container
#COPY . .

RUN go get ./...


# Make port 8080 available to the host
EXPOSE 8080

# Build and run the server when the container is started
#RUN go build /app/server.go
#ENTRYPOINT ./server

RUN air init

CMD ["air"]
