class Brr < Formula
  desc "Terminal speed reading tool using RSVP technique"
  homepage "https://github.com/metcalfc/brr"
  version "0.1.4"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/metcalfc/brr/releases/download/v0.1.4/brr_v0.1.4_darwin_amd64.tar.gz"
      sha256 "74035e057ae5067d3082807ee25246f5d250d248a9bed1a500e148a50e3751fb"
    end
    on_arm do
      url "https://github.com/metcalfc/brr/releases/download/v0.1.4/brr_v0.1.4_darwin_arm64.tar.gz"
      sha256 "49172a9a065a14299d5b9aabd49c51f7936d5e2100e3f1c08cd42eb9651e4cb6"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/metcalfc/brr/releases/download/v0.1.4/brr_v0.1.4_linux_amd64.tar.gz"
      sha256 "2b21d1e890fce608d6c2e36784f703e5dc6a25987ea5da3c533b890d80d92522"
    end
    on_arm do
      url "https://github.com/metcalfc/brr/releases/download/v0.1.4/brr_v0.1.4_linux_arm64.tar.gz"
      sha256 "a6d2acd99b0fd7451468f016951f42b62ff1252873e5e326b88f10da475ecc51"
    end
  end

  def install
    bin.install "brr"
  end

  test do
    assert_match "brr 0.1.4", shell_output("#{bin}/brr -v")
  end
end
