# go-rabbitmq
Go implementation for queue with rabbitmq

1). make rabbitmq (will build docker for rabbitmq server)
2). make run (open a new terminal; will build the app and run it)
3). make consumers (open a new termianl; will run cli command to run long running consumers)

after this you can login to localhost:15672 and login with quest:quest to the rabbitmq admin panel.
open brower to localhost:8080/api to run/dispatch a job to the queue. consumers should pick up the job and should be visiable in the rabbitmq panel.
