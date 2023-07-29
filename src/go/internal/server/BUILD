go_package(
    test_extra_env_vars=[
        f"CONFIG_DIR=/Users/ianarmstrong/Repos/distributed-services-with-go/certs",
        f"ACL_DIR=/Users/ianarmstrong/Repos/distributed-services-with-go/src/go/tests",
        "HOME",
    ],
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
