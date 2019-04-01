#!/bin/bash
docker-compose up -d
docker-compose exec  scylla /bin/bash
sudo chown -R $USER:$USER .
