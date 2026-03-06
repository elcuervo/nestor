{
  description = "Nestor - Network Share via TOR";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      # Systems that have embed_*.go files in the source
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-darwin" ];
      allSystems = supportedSystems ++ [ "aarch64-linux" ];

      forAllSystems = systems: f: nixpkgs.lib.genAttrs systems (system: f system);

      torBinariesInfo = {
        "x86_64-linux"   = { torBinDir = "linux-amd64";  libName = "libevent-2.1.so.7"; };
        "x86_64-darwin"  = { torBinDir = "darwin-amd64"; libName = "libevent-2.1.7.dylib"; };
        "aarch64-darwin" = { torBinDir = "darwin-arm64"; libName = "libevent-2.1.7.dylib"; };
      };
    in {
      packages = forAllSystems supportedSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          info = torBinariesInfo.${system};
        in {
          default = pkgs.buildGoModule {
            pname = "nestor";
            version = "0.1.2";
            src = ./.;

            vendorHash = "sha256-aSWvssGJHQp6yIDjiAC7vlVXM0Q9OOQuPyMMO4+t2MA=";

            nativeBuildInputs = [ pkgs.xz ];

            preBuild = ''
              mkdir -p tor_binaries/${info.torBinDir}

              cp ${pkgs.tor}/bin/tor tor_binaries/${info.torBinDir}/tor
              chmod +x tor_binaries/${info.torBinDir}/tor
              xz -9 tor_binaries/${info.torBinDir}/tor

              cp ${pkgs.libevent}/lib/${info.libName} tor_binaries/${info.torBinDir}/${info.libName}
              xz -9 tor_binaries/${info.torBinDir}/${info.libName}
            '';
          };
        });

      apps = forAllSystems supportedSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/nestor";
        };
      });

      devShells = forAllSystems allSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              curl
              tor
              gotools
              go-tools
            ];
          };
        });
    };
}
