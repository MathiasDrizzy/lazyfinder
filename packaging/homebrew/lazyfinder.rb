# Homebrew formula for lazyfinder. Copy it to a tap (homebrew-tap/Formula/lazyfinder.rb) and fill in the
# version and the sha256 of each archive from the release's checksums.txt.
class Lazyfinder < Formula
  desc "Finder-like file manager for the terminal: columns, preview, open-with, live follow"
  homepage "https://github.com/MathiasDrizzy/lazyfinder"
  version "0.7.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/MathiasDrizzy/lazyfinder/releases/download/v#{version}/lazyfinder_#{version}_darwin_arm64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_OF_darwin_arm64"
    end
    on_intel do
      url "https://github.com/MathiasDrizzy/lazyfinder/releases/download/v#{version}/lazyfinder_#{version}_darwin_amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_OF_darwin_amd64"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/MathiasDrizzy/lazyfinder/releases/download/v#{version}/lazyfinder_#{version}_linux_arm64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_OF_linux_arm64"
    end
    on_intel do
      url "https://github.com/MathiasDrizzy/lazyfinder/releases/download/v#{version}/lazyfinder_#{version}_linux_amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_OF_linux_amd64"
    end
  end

  def install
    bin.install "lazyfinder"
    man1.install "docs/lazyfinder.1"
    generate_completions_from_executable(bin/"lazyfinder", "--completions", shells: [:bash, :zsh, :fish])
  end

  test do
    assert_match "lazyfinder", shell_output("#{bin}/lazyfinder --version")
  end
end
