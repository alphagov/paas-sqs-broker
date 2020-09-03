argsOuter@{...}:
let
  # specifying args defaults in this slightly non-standard way to allow us to include the default values in `args`
  args = rec {
    pkgs = import <nixpkgs> {};
    go = pkgs.go;
    localOverridesPath = ./local.nix;
  } // argsOuter;
in (with args; {

  paasSqsBrokerEnv = (pkgs.stdenv.mkDerivation rec {
    name = "paas-sqs-broker-env";
    shortName = "sqsbrk";
    buildInputs = with pkgs; [ gitFull cacert go ];

    LD_LIBRARY_PATH = "${pkgs.stdenv.lib.makeLibraryPath buildInputs}";
    LANG="en_GB.UTF-8";
    GOPATH = (toString (./.)) + "/.gopath";
    GOFLAGS = "-mod=vendor";

    shellHook = ''
      export PS1="\[\e[0;36m\](nix-shell\[\e[0m\]:\[\e[0;36m\]${shortName})\[\e[0;32m\]\u@\h\[\e[0m\]:\[\e[0m\]\[\e[0;36m\]\w\[\e[0m\]\$ "

      mkdir -p $GOPATH
    '';
  }).overrideAttrs (if builtins.pathExists localOverridesPath then (import localOverridesPath args) else (x: x));
})
