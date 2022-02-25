#!/bin/sh

pts_bin=/root/phoronix-test-suite/phoronix-test-suite

run_natively() {
  local test_suite="$1"
  
  ${pts_bin} batch-run ${test_suite}
}

run_in_tmux() {
  local test_suite="$1"

  unbuffer tmux new-session -n pts \
    "${pts_bin} batch-run '${test_suite}'"
}

publish_results() {
  # Write a config map with the results.
  pts-write-cm
  # As a fallback option, publish the results via Pod logs.
  cat /var/lib/phoronix-test-suite/test-results/*/composite.xml
}

main() {
  local test_suite="$1"

  if test "$test_suite"; then
    run_natively "$test_suite"
    publish_results
  else
    # Give the user the option to debug/run tests interactively.
    sleep inf
  fi
}

main "$@"
