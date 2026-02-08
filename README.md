# TSS wrapper


The service is responsible for managing the TSS daemon:
- starting the TSS daemon 
- listen for update events from the Bridgeless-core
- schedule the TSS update when we need it






On the start of service it makes the following steps:
- Runs the TSS daemon 
- 