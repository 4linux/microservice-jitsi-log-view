# microservice-jitsi-log-view

Microservice para visualizacao do logging de eventos de login e logout do Jitsi. 

## TODOs

- [ ] return error to client
- [ ] return error if search doesnt bring any results
- [ ] Treat possible invalid or null query params (maybe a parser function)
- [ ] Error handling for size of the dataset
- [ ] Accept wildcard in search
- [ ] Add w at find functions (find functions respond request directly)
- [ ] Unit Tests
- [ ] search about CORS
- [ ] Summary with presence time (Diff between login/logout)
- [ ] Log requests
- [ ] regex for email validator (search about attacks based on input injection)
- [ ] Convert to string from MongoDB in case of data with divergent type
- [ ] Select only a few fields
- [ ] REFACT IT

## Agradecimentos

* [Arthur Nascimento](https://github.com/tureba) - MongoDB
* [Hector Vido](https://github.com/hector-vido) - MongoDB
* [Júlio Rangel Ballot](https://github.com/jrballot) - MongoDB