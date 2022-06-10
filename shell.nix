let
  pkgs = import
    (builtins.fetchTarball {
      url = "https://github.com/NixOS/nixpkgs/archive/6d02a514db95d3179f001a5a204595f17b89cb32.tar.gz";
      sha256 = "1siqqsxvl31c4yz0cizdg2irbfzhi033rny1sjs52jc5hxi54yv9";
    })
    { };
in
pkgs.mkShell {
  buildInputs = with pkgs; [
    go
  ];
}
