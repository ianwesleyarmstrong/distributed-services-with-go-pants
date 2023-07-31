go_package(dependencies=[":certs", ":auth"], test_extra_env_vars=["DEBUG=true"])

relocated_files(
    name="certs",
    files_targets=[
        "certs:init-gen-cert",
        "certs:gen-cert-root-client",
        "certs:gen-cert-nobody-client",
        "certs:gen-cert-server",
    ],
    src="certs",
    dest="src/go/internal/agent",
)

relocated_files(
    name="auth",
    files_targets=["src/go/tests:conf", "src/go/tests:policy"],
    src="src/go/tests",
    dest="src/go/internal/agent",
)
