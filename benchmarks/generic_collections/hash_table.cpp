#include <cstdint>
#include <cstdio>
#include <vector>

static std::int32_t lookup_or(const std::vector<std::int32_t>& keys,
                              const std::vector<std::int32_t>& values,
                              std::int32_t key,
                              std::int32_t fallback) {
  for (std::size_t i = 0; i < keys.size() && i < values.size(); ++i) {
    if (keys[i] == key) {
      return values[i];
    }
  }
  return fallback;
}

int main() {
  constexpr std::int32_t n = 1024;
  std::vector<std::int32_t> keys(n);
  std::vector<std::int32_t> values(n);
  for (std::int32_t i = 0; i < n; ++i) {
    keys[static_cast<std::size_t>(i)] = i * 2 + 1;
    values[static_cast<std::size_t>(i)] = i + 7;
  }

  std::int32_t checksum = 0;
  for (std::int32_t q = 0; q < n; ++q) {
    checksum += lookup_or(keys, values, q * 2 + 1, 0);
  }
  std::printf("%d\n", checksum);
  return checksum == 530944 ? 0 : 1;
}
