gcloud functions deploy GoogleChatAlert \
--project=<your project id> \
--region <your project region> \
--entry-point GoogleChatAlert \
--runtime go119 \
--trigger-topic alert
