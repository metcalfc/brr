class Brr < Formula
  desc "Terminal speed reading tool using RSVP technique"
  homepage "https://github.com/metcalfc/brr"
  url "https://github.com/metcalfc/brr/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "" # Will be filled in after creating the release
  license "MIT"
  head "https://github.com/metcalfc/brr.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")
    man1.install "brr.1"
  end

  test do
    # Test that the binary exists and shows help
    assert_match "Brr - Terminal Speed Reading Tool", shell_output("#{bin}/brr -h", 1)

    # Test with sample text
    (testpath/"test.txt").write("Hello world test")
    # Can't fully test interactive features in Homebrew test environment
    # but we can verify the binary accepts the file
    assert_match "test.txt", shell_output("#{bin}/brr #{testpath}/test.txt 2>&1", 1)
  end
end
