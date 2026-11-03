{
  description = "Entorno de desarrollo para el motor Go de go_htmx_crud";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forEachSystem = nixpkgs.lib.genAttrs supportedSystems;
    in
    {
      devShells = forEachSystem (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            # Paquetes necesarios en el entorno de desarrollo
            packages = with pkgs; [
              go
              sqlite
              sqlc # Generate type-safe code from SQL
              golangci-lint # Fast linters Runner for Go
            ];

            shellHook = ''
              echo "Entorno de desarrollo para seb-harness activado."
              echo "Herramientas disponibles: go $(go version | awk '{print $3}'), sqlite3, sqlc $(sqlc version), golangci-lint."
              echo ""
            '';
          };
        }
      );
    };

}
