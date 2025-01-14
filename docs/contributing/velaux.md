# Contribute VelaUX

## Prepare the local environment

### Install VelaCore

1. Check requirements

    * VelaD support installing KubeVela on machines based on these OS: Linux, macOS, Windows.
    * If you are using Linux or macOS, make sure your machine have `curl` installed.
    * If you are using macOS or Windows, make sure you've already installed [Docker](https://www.docker.com/products/docker-desktop).

2. Download the binary.

    * MacOS/Linux

    ```bash
    curl -fsSl https://static.kubevela.net/script/install-velad.sh | bash
    ```

    * Windows

    ```bash
    powershell -Command "iwr -useb https://static.kubevela.net/script/install-velad.ps1 | iex"
    ```

3. Install

    ```bash
    velad install
    ```

4. Install VelaUX environment

    ```bash
    vela addon enable ./addon
    ```

## Start the server on local

Make sure you have installed [yarn](https://classic.yarnpkg.com/en/docs/install).

```shell
yarn install

## Build the frontend and watch the code changes
yarn dev
```

### Start the server

```shell
## Setting the KUBECONFIG environment
export KUBECONFIG=$(velad kubeconfig --host)

make run-server
```

Waiting the server started, open http://127.0.0.1:8000 via the browser.

Now, the local environment is built successfully, you could write the server or frontend code.

Notes:

* If you change the frontend code, it will take effect after the website refresh.
* If you change the server code, it will take effect after restarted the server.

### Check the code style

```shell
# Frontend
yarn lint
# Server
make reviewable
```

### Test the code

Frontend:

```shell
yarn test
```

Server:

```shell
make unit-test-server
make e2e-server-test
```

### Generate the OpenAPI schema

```shell
make build-swagger
```

## References

* UI framework: [@alifd/next](https://fusion.design/)
* Icons: [react-icons](https://react-icons.github.io/react-icons)
