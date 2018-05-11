FROM iron/go:dev
RUN mkdir /app
COPY src/main/main.go /app/
ENV SRC_DIR=/app
ADD . $SRC_DIR
RUN go get goji.io
RUN go get gopkg.in/mgo.v2
# RUN cd $SRC_DIR; go build -o main
CMD ["go", "run", "/app/main.go"]