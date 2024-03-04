podman run -it --env-file $(pwd)/.env --volume $(pwd)/data:/app/data:Z -p 5000:5000 go-attendence
