file(name="ca-csr", source="ca-csr.json")
file(name="ca-config", source="ca-config.json")
file(name="server-csr", source="server-csr.json")

system_binary(
    name="cfssl",
    binary_name="cfssl",
    # fingerprint=r"1.6.4",
    # fingerprint_args=["version"]
)

run_shell_command(
    name="init-gen-cert",
    execution_dependencies=[":ca-csr", ":ca-config", ":server-csr"],
    runnable_dependencies=[":cfssl"],
    command="""
    cfssl gencert \
        -initca ca-csr.json | cfssljson -bare ca
    """
)

run_shell_command(
    name="gen-cert",
    execution_dependencies=[":ca-csr", ":ca-config", ":server-csr", ":init-gen-cert"],
    runnable_dependencies=[":cfssl"],
    command="""    
    cfssl gencert \
        -ca=ca.pem \
        -ca-key=ca-key.pem \
        -config=ca-config.json \
        -profile=server \
        server-csr.json | cfssljson -bare server
    """
)