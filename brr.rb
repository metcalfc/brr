class Brr < Formula
  desc "Terminal speed reading tool using RSVP technique"
  homepage "https://github.com/metcalfc/brr"
  version "0.1.2"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/metcalfc/brr/releases/download/v0.1.2/brr_v0.1.2_darwin_amd64.tar.gz"
      sha256 "891b0aaeecc2ad768917a23e03437dd44f66cb975b299eb60bdf119590af14bb"
    end
    on_arm do
      url "https://github.com/metcalfc/brr/releases/download/v0.1.2/brr_v0.1.2_darwin_arm64.tar.gz"
      sha256 "d2286ffca42221634205934b26be1611501748c1c301986b3a899897cccff4a1"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/metcalfc/brr/releases/download/v0.1.2/brr_v0.1.2_linux_amd64.tar.gz"
      sha256 "4dd1542c9658e316c897100a3e663cc9c31693f8df55e9ffd8bb5b80a3c702ae"
    end
    on_arm do
      url "https://github.com/metcalfc/brr/releases/download/v0.1.2/brr_v0.1.2_linux_arm64.tar.gz"
      sha256 "6b7beb9db6136b9edb991f839bfcc0b83cefa8949d79c8313dffa0c3b4710556"
    end
  end

  def install
    bin.install "brr"
  end

  test do
    assert_match "brr 0.1.2", shell_output("#{bin}/brr -v")
  end
end
