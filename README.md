# symctl

## Installation

To install `symctl` run the following command:
```bash
bash <(curl -s https://raw.githubusercontent.com/SymmetricalAI/symctl/main/install/install.sh)
```

... and then make sure to add the following to your `~/.bashrc` or `~/.zshrc`:

```bash
export PATH=$PATH:$HOME/.symctl/bin
```

## Docker Image:

- Docker image is built for symctl
- Its running in port 8080
- Tag the docker image by your docker-hub username
 ```bash
   docker tag symctl your-dockerhub-username/symctl
```
- Push the tagged image to your docker-hub repository using the following command

```bash
    docker push your-dockerhub-username/symctl
```