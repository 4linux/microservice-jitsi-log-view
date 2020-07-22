# microservice-jitsi-log-view

Microservice para visualizacao do logging de eventos de login e logout do Jitsi. 

## TODOs

- [ ] return error to client when something goes wrong instead of 200, nil
- [ ] return error if search doesnt bring any results
- [ ] Treat possible invalid or null query params (maybe a parser function)
- [ ] Add w at find functions
- [ ] Summary with presence time (Diff between login/logout)
- [ ] Log requests
- [ ] Unit Tests
- [ ] search about CORS
- [ ] Convert to string from MongoDB in case of data with divergent type
- [ ] Input sanitize
- [ ] REFACT IT

## Agradecimentos

* [Arthur Nascimento](https://github.com/tureba) - MongoDB
* [Hector Vido](https://github.com/hector-vido) - MongoDB
* [Júlio Rangel Ballot](https://github.com/jrballot) - MongoDB