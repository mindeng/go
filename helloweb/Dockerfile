#FROM scratch
#COPY ./main /main
#EXPOSE 80
#ENTRYPOINT ["/main"]
FROM golang:alpine as build
RUN apk --no-cache add tzdata
WORKDIR /app
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main
FROM scratch as final
COPY --from=build /app/main .
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ 'Asia/Shanghai'
EXPOSE 80
ENTRYPOINT ["/main"]