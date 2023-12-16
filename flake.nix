{
  description = "Automate Kubernetes Configuration Editing";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts.url = "github:hercules-ci/flake-parts";

    gomod2nix.url = "github:nix-community/gomod2nix";
    gomod2nix.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {

      systems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin" ];

      perSystem = { pkgs, inputs', ... }: {

        packages.default = inputs'.gomod2nix.legacyPackages.buildGoApplication rec {
          pname = "kpt";

          version = with inputs; "${self.shortRev or self.dirtyShortRev or "dirty"}";

          src = inputs.self;

          modules = "${inputs.self}/gomod2nix.toml";

          subPackages = [ "." ];

          ldflags = [
            "-s" "-w"
            "-X github.com/GoogleContainerTools/kpt/run.version=git-${version}"
          ];

          meta = {
            homepage = "https://github.com/kptdev/kpt";
          };
        };

        packages.gomod2nix = inputs'.gomod2nix.packages.default;
      };
    };
}
