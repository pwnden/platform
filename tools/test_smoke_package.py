"""Guard answer extraction at the trusted verification boundary."""

import unittest

from smoke_package import verified_answer


class VerificationAnswerTests(unittest.TestCase):
    def test_verification_result_supplies_submission_fixture(self):
        self.assertEqual(verified_answer('verified rotor-lock: pwnden{example}\n', 'rotor-lock'),
                         'pwnden{example}')
        self.assertEqual(verified_answer('verified note-vault: pwnden{example}\n'
                                         'patched attack failed; functional check passed\n', 'note-vault'),
                         'pwnden{example}')

    def test_wrong_problem_diagnostics_and_multiple_answers_are_rejected(self):
        for output in ('verified note-vault: pwnden{example}', 'pwnden{example}',
                       'error: pwnden{example}', 'verified rotor-lock: pwnden{one}\npwnden{two}',
                       'verified rotor-lock: ', 'verified rotor-lock: pwnden{bad value}'):
            with self.subTest(output=output), self.assertRaises(ValueError):
                verified_answer(output, 'rotor-lock')


if __name__ == '__main__':
    unittest.main()
