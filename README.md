
- setup k8s cluster : <br>
  🧪(Here is example of setting podman, K8s cluster and WSL ➡️ [Link](https://github.com/shivaprasadmb/wsl-podman-k8s))
- Build image using containerfile, use proper tag so that the same name will be used in deployment.yml file
    ```
    docker/podman build -t <image:tag> -f containerfile
    ```

- Steps to generate cert keys for webhoob server, do it in your local system.

    1. create private key for CA
    ```
    openssl genrsa -out ca.key 2048
    ```
    2. create self signed cert for CA, the CN field should be same as name of webhook k8s service object
    ```
    openssl req -x509 -new -nodes -key ca.key -subj "/CN=my-webhook-ca" -days 3650 -out ca.crt
    ```
    3. generate private key for webhook server
    ```
    openssl genrsa -out webhook-server.key 2048
    ```
    openssl.cnf
    ```
    [req]
    distinguished_name = req_distinguished_name
    req_extensions = v3_req
    prompt = no

    [req_distinguished_name]
    CN = validating-webhook-service.webhooks.svc

    [v3_req]
    subjectAltName = @alt_names

    [alt_names]
    DNS.1 = validating-webhook-service
    DNS.2 = validating-webhook-service.webhooks
    DNS.3 = validating-webhook-service.webhooks.svc
    ```
    4. create cert signing request for wehbook server using openssl.cnf SAN extension
    ```
    openssl req -new -key tls.key -out webhook-server.csr -config openssl.cnf
    ```
    5. sign cert usng CA cert
    ```
    openssl x509 -req -in webhook-server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out webhook-server.crt -days 365 -extensions v3_req -extfile openssl.cnf
    ```

- create k8s secret of tls type, this will be used for mounting certs onto running pod
    ```
    k create secret tls validating-webhook-certs --cert=path/to/webhook-server.crt --key=path/to/webhook-server.key -n webhooks
    ```
- encode CA cert file that will be used in webhook config file
    ```
    cat webhook-certs/ca.crt | base64 | tr -d '\n'
    ```
- apply k8s objects
    ```
    k apply -f deplyment.yml
    k apply -f service.yml
    k apply -f webhook-config.yml
    ```
    
- test running a pod without app label, it should call webhook server and deny request
    ```
    k run nginx --image=nginx
    ```
