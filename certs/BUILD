file(name="ca-csr", source="ca-csr.json")
file(name="ca-config", source="ca-config.json")
file(name="server-csr", source="server-csr.json")
file(name="client-csr", source="client-csr.json")

shell_command(
    name="init-gen-cert",
    execution_dependencies=[":ca-csr", ":ca-config", ":server-csr"],
    tools=["cfssl", "cfssljson"],
    command="""
    cfssl gencert \
        -initca ca-csr.json | cfssljson -bare ca
    """,
    output_files=["ca.pem", "ca-key.pem", "ca.csr"],
)

shell_command(
    name="gen-cert-server",
    log_output=True,
    execution_dependencies=[":ca-csr", ":ca-config", ":server-csr", ":init-gen-cert"],
    tools=["cfssl", "cfssljson"],
    command="""    
    cfssl gencert \
        -ca=ca.pem \
        -ca-key=ca-key.pem \
        -config=ca-config.json \
        -profile=server \
        server-csr.json | cfssljson -bare server
    """,
    output_files=["server.pem", "server-key.pem", "server.csr"],
)

shell_command(
    name="gen-cert-root-client",
    log_output=True,
    execution_dependencies=[":ca-csr", ":ca-config", ":client-csr", ":init-gen-cert"],
    tools=["cfssl", "cfssljson", "ls"],
    command="""    
    cfssl gencert \
        -ca=ca.pem \
        -ca-key=ca-key.pem \
        -config=ca-config.json \
        -profile=client \
        -cn="root" \
        client-csr.json | cfssljson -bare root-client
    """,
    output_files=["root-client.pem", "root-client-key.pem", "root-client.csr"],
)

shell_command(
    name="gen-cert-nobody-client",
    log_output=True,
    execution_dependencies=[":ca-csr", ":ca-config", ":client-csr", ":init-gen-cert"],
    tools=["cfssl", "cfssljson"],
    command="""    
    cfssl gencert \
        -ca=ca.pem \
        -ca-key=ca-key.pem \
        -config=ca-config.json \
        -profile=client \
        -cn="nobody" \
        client-csr.json | cfssljson -bare nobody-client
    """,
    output_files=["nobody-client.pem", "nobody-client-key.pem", "nobody-client.csr"],
)
