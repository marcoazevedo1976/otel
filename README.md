# Desafio OpenTelemetry | Pós Go Expert

## Instruções
1. Clone o repositório
2. Entre na pasta do repositório
3. Execute: docker compose up --build -d
4. Execute um POST em http://localhost:8080/cep com o schema abaixo

    ```json
    {
    "cep": "78048250"
    }
    ```
    Se tiver a extenção do VSCode "REST Client", pode executar o arquivo "api.http"  

5. Acesse o Zipkin em http://localhost:9411