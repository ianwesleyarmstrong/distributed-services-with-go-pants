go_package(
    dependencies=[":certs", ":auth", ":debug"],
)

shell_command(
    name="debug",
    execution_dependencies=[":certs", ":auth"],
    tools=["ls"],
    command="""
    ls -latr
    """,
    log_output=True,
)

relocated_files(
    name="certs",
    files_targets=[
        "certs:init-gen-cert",
        "certs:gen-cert-root-client",
        "certs:gen-cert-nobody-client",
        "certs:gen-cert-server",
    ],
    src="certs",
    dest="src/go/internal/server",
)

relocated_files(
    name="auth",
    files_targets=["src/go/tests:conf", "src/go/tests:policy"],
    src="src/go/tests",
    dest="src/go/internal/server",
)
