# Add new testcase files to this directory

## Path

Fixtures are grouped by the major version of the core standard: `v0/` and
`v1/`, not by minor version (such as `v0.2/`). Paths below are relative to
this directory.

The tests in [`v0_test.go`](../v0_test.go) read these directories:

* `v0/valid/`: Valid publiccode.yml files without warnings.
* `v0/valid_with_warnings/`: Valid files with expected warnings.
* `v0/invalid/`: Files with expected validation errors.

Each directory also has a `no-network/` subdirectory. Put cases that test
parsing with `ParserConfig.DisableNetwork` enabled there.

The tests in [`v1_test.go`](../v1_test.go) read `v1/invalid/`,
`v1/invalid_valid_in_v0/` and `v1/invalid_valid_with_warnings_in_v0/`.
They currently expect the same unsupported version error for every file.
They do not validate v1 fields. `v1/valid/` is not read by these tests.

The globs only read YAML files directly inside each listed directory.
The v0 `no-network/` directories have separate tests. The v1 tests do not
read their `no-network/` subdirectories.

## File contents and expected results

Start from [`v0/valid/valid.minimal.yml`](v0/valid/valid.minimal.yml) and
edit it to contain the subject of your test. Keep `publiccodeYmlVersion`
consistent with the version you want to test.

For every new file in `v0/invalid/` or `v0/valid_with_warnings/`, add an
entry keyed by its filename to the `expected` map in the corresponding
test in `v0_test.go`. The same applies to their `no-network/` subdirectories:
use the corresponding `_NoNetwork` test. Record the expected diagnostics,
including their line and column numbers. Without an entry, the test fails
with "No expected data for file". Valid files without warnings are discovered
automatically and need no map entry.

Add comments only to the section / key being tested to draw attention
to it and, if the testcase needs more context, add it as a comment at
the top of the file.

## Filename

### Testcases validating keys

Filenames for testcases testing a particular key follow this format:

`KEYNAME_REASON.yml`

Where `KEYNAME` is the name of the key being checked in the test (eg. `name`).
Use an underscore as a separator for nested keys (eg. `maintenance_contracts`).

`REASON` is the scenario the testcase is checking. It should be descriptive and
short. Good `REASON`s are:

* `nil`: The key is nil
* `empty`: The key is an empty string
* `missing`: The key is missing
* `invalid`: Generic, the key format is wrong.

Examples (not necessarly real):

* `valid/key_nil.yml`: We expect the key to be nil
* `invalid/maintenance_contracts_empty.yml`: We expect maintenance/contracts key
  to contain something.
* `invalid/longDescription_invalid.yml`: We are testing an unspecified invalid
  value in longDescription.
* `invalid/longDescription_blocked_word.yml`: We expect longDescription doesn't contain
  any blocked word.

### Other testcases

Use a short and descriptive name for testcases not focusing on
a particular key, eg `file_encoding.yml`
