class Kecs < Formula
  desc "Kubernetes-based ECS Compatible Service"
  homepage "https://github.com/nandemo-ya/kecs"
  license "Apache-2.0"

  # Stable release version
  stable do
    version "0.1.0"
    
    on_macos do
      if Hardware::CPU.intel?
        url "https://github.com/nandemo-ya/kecs/releases/download/v0.1.0/kecs_v0.1.0_Darwin_x86_64.tar.gz"
        sha256 "PLACEHOLDER_SHA256_DARWIN_AMD64"
      else
        url "https://github.com/nandemo-ya/kecs/releases/download/v0.1.0/kecs_v0.1.0_Darwin_arm64.tar.gz"
        sha256 "PLACEHOLDER_SHA256_DARWIN_ARM64"
      end
    end

    on_linux do
      if Hardware::CPU.intel?
        url "https://github.com/nandemo-ya/kecs/releases/download/v0.1.0/kecs_v0.1.0_Linux_x86_64.tar.gz"
        sha256 "PLACEHOLDER_SHA256_LINUX_AMD64"
      else
        url "https://github.com/nandemo-ya/kecs/releases/download/v0.1.0/kecs_v0.1.0_Linux_arm64.tar.gz"
        sha256 "PLACEHOLDER_SHA256_LINUX_ARM64"
      end
    end
  end

  # Development/pre-release version (optional)
  # Users can install with: brew install kecs --HEAD
  head do
    url "https://github.com/nandemo-ya/kecs.git", branch: "main"
    
    depends_on "go" => :build
  end

  def install
    if build.head?
      # Build from source for HEAD version
      system "go", "build", 
             "-ldflags", "-s -w -X github.com/nandemo-ya/kecs/controlplane/internal/controlplane/cmd.Version=HEAD",
             "-o", bin/"kecs",
             "./controlplane/cmd/controlplane"
    else
      # Install pre-built binary for stable version
      bin.install "kecs"
    end
  end

  def post_install
    # Create config directory
    (etc/"kecs").mkpath
  end

  def caveats
    <<~EOS
      KECS has been installed! Here's how to get started:

      1. Start KECS in a container:
         kecs start

      2. Check status:
         kecs status

      3. View help:
         kecs --help

      Requirements:
      - Docker or a Kubernetes cluster
      - For local development: k3d or kind

      Configuration:
      - Config directory: #{etc}/kecs

      Documentation:
      - https://github.com/nandemo-ya/kecs
    EOS
  end

  test do
    # Test version command
    assert_match version.to_s, shell_output("#{bin}/kecs version 2>&1")
    
    # Test help command
    assert_match "Kubernetes-based ECS Compatible Service", shell_output("#{bin}/kecs --help 2>&1")
  end
end

# For pre-release versions, create a separate formula:
# kecs-beta.rb or kecs@beta.rb
class KecsBeta < Formula
  desc "Kubernetes-based ECS Compatible Service (Beta)"
  homepage "https://github.com/nandemo-ya/kecs"
  license "Apache-2.0"
  version "1.0.0-beta.1"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/nandemo-ya/kecs/releases/download/v1.0.0-beta.1/kecs_v1.0.0-beta.1_Darwin_x86_64.tar.gz"
      sha256 "PLACEHOLDER"
    else
      url "https://github.com/nandemo-ya/kecs/releases/download/v1.0.0-beta.1/kecs_v1.0.0-beta.1_Darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/nandemo-ya/kecs/releases/download/v1.0.0-beta.1/kecs_v1.0.0-beta.1_Linux_x86_64.tar.gz"
      sha256 "PLACEHOLDER"
    else
      url "https://github.com/nandemo-ya/kecs/releases/download/v1.0.0-beta.1/kecs_v1.0.0-beta.1_Linux_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  conflicts_with "kecs", because: "both install kecs binary"

  def install
    bin.install "kecs"
  end

  test do
    system "#{bin}/kecs", "version"
  end
end