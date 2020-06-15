with import <nixpkgs> {};

stdenv.mkDerivation {
  name = "go-dev-env";

  buildInputs = [
    pkgs.figlet
    pkgs.lolcat
    pkgs.go
    pkgs.jq
    pkgs.curl
    pkgs.httpie
  ];

  shellHook = ''
    figlet "Bem vindo!" | lolcat --freq 0.5
    export GOPATH="$(pwd)/.go"
    export GOCACHE=""
    export GO111MODULE='on'
  '';
}
