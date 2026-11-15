"""Guard tests for the bounded DOCX closeout and suite decision patch."""
import copy
import hashlib
import json
import tempfile
import unittest
from pathlib import Path
import closeout_dcm34_docx as patch


def fixture():
    complete = [f'DCM-{i:02}' for i in range(1, 34) if i != 19]
    todo = {
        'schema_version': 1, 'app': 'DCM QuickSort', 'suite': 'FileFlow Suite',
        'scope': 'Independent native application',
        'completion_plan': {'step_count': 50, 'completed_setup_steps': complete[:5],
            'completed_product_steps': complete[5:], 'next_step': 'DCM-34',
            'next_runnable_step': 'DCM-34', 'blocked_steps': ['DCM-19']},
        'checkpoints': {'historical_receipt': {'passed': False, 'commit': 'old'}},
        'evidence': {'dcm34_docx_fallback_probe': 'original-probe.json'},
        'items': [
            {'id': 'DCM-34-DOCX-FALLBACK-01', 'status': 'planned',
             'ui_test': 'not_run', 'user_test': 'not_recorded',
             'existing_evidence': 'Original RED must remain unchanged',
             'evidence_refs': ['dcm34_docx_fallback_probe']},
            {'id': 'DCM-RELEASE-01', 'status': 'planned', 'blocked_by': ['D1']},
            {'id': 'D1', 'status': 'tested', 'original': {'keep': True}}]},
    todo = todo[0]
    plan = {'steps': [{'id': f'DCM-{i:02}',
             'completion': 'complete' if f'DCM-{i:02}' in complete else ('in_progress' if i == 34 else 'pending'),
             'status': 'implemented' if i == 34 else 'planned',
             'user_test': 'not_recorded', 'evidence': [],
             **({'native_test': {'source': 'old-pdf'}, 'qualification_note': 'Original PDF milestone'} if i == 34 else {})}
             for i in range(1, 51)], 'next_step': 'DCM-34'}
    return {
        'CONTEXT.md': '# DCM QuickSort context\n\n## Current checkpoint\nHistorical material stays byte-identical.\n',
        'todo.json': json.dumps(todo, indent=2) + '\n',
        'qualification/completion-50.json': json.dumps(plan, indent=2) + '\n',
        'qualification/dcm34-preview-failures.md': '# DCM-34\n\n## Separate confirmed DOCX fallback defect\nOriginal open/unfixed observation.\n\n## Remaining DCM-34 scope\nPDF cases remain.\n',
    }


class CloseoutTests(unittest.TestCase):
    def test_blank_refused(self):
        with self.assertRaises(ValueError):
            patch.transform({})

    def test_error_no_writes(self):
        docs = fixture()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name, text in docs.items():
                f = root / name
                f.parent.mkdir(parents=True, exist_ok=True)
                f.write_text(text, encoding='utf-8')
            before = {name: (root / name).read_bytes() for name in docs}
            with self.assertRaises(ValueError):
                patch.apply(root)  # synthetic inputs must not match exact source guards
            self.assertEqual(before, {name: (root / name).read_bytes() for name in docs})

    def test_success_preserves_history(self):
        docs = fixture()
        out = patch.transform(docs)
        old = json.loads(docs['todo.json'])
        new = json.loads(out['todo.json'])
        self.assertEqual(old['completion_plan'], new['completion_plan'])
        self.assertEqual(old['checkpoints']['historical_receipt'], new['checkpoints']['historical_receipt'])
        self.assertEqual(old['items'][2], new['items'][2])
        self.assertEqual(old['items'][0]['existing_evidence'], new['items'][0]['existing_evidence'])
        self.assertEqual('fixed', new['items'][0]['status'])
        self.assertEqual('not_recorded', new['items'][0]['user_test'])
        self.assertTrue(out['CONTEXT.md'].endswith(docs['CONTEXT.md'].split('\n', 1)[1]))
        self.assertIn('Original open/unfixed observation.', out['qualification/dcm34-preview-failures.md'])

    def test_parent_not_closed(self):
        out = patch.transform(fixture())
        plan = json.loads(out['qualification/completion-50.json'])
        self.assertEqual(50, len(plan['steps']))
        self.assertEqual(32, sum(s['completion'] == 'complete' for s in plan['steps']))
        self.assertEqual('in_progress', plan['steps'][33]['completion'])
        self.assertEqual({'source': 'old-pdf'}, plan['steps'][33]['native_test'])
        self.assertEqual('pending', plan['steps'][18]['completion'])

    def test_suite_gate_does_not_block_development(self):
        out = patch.transform(fixture())
        todo = json.loads(out['todo.json'])
        gate = todo['suite_release_gate']
        self.assertEqual(['Flow Search', 'DeepSearch'], gate['requires_ready'])
        self.assertFalse(gate['blocks_component_development_or_focused_tests'])
        self.assertIn('FILEFLOW-SUITE-READY-01', todo['items'][1]['blocked_by'])
        plan = json.loads(out['qualification/completion-50.json'])
        for s in plan['steps']:
            self.assertEqual(int(s['id'][-2:]) >= 47, 'suite_release_gate' in s)

    def test_duplicate_or_missing_target_refused(self):
        for duplicate in (True, False):
            docs = fixture()
            data = json.loads(docs['todo.json'])
            if duplicate:
                data['items'].append(copy.deepcopy(data['items'][0]))
            else:
                data['items'].pop(0)
            docs['todo.json'] = json.dumps(data)
            with self.assertRaises(ValueError):
                patch.transform(docs)

    def test_already_applied_refused(self):
        once = patch.transform(fixture())
        with self.assertRaises(ValueError):
            patch.transform(once)

    def test_no_false_acceptance_or_publication(self):
        out = patch.transform(fixture())
        data = json.loads(out['qualification/evidence/dcm-34-docx-fallback-native-summary.json'])
        self.assertEqual('not_recorded', data['user_test'])
        self.assertEqual('PARTIAL', data['parent_step_status'])
        self.assertFalse(data['new_native_run_claimed'])
        self.assertEqual(21, data['separate_visual_review']['images_reviewed'])
        self.assertFalse(data['separate_visual_review']['raw_flags_modified'])
        self.assertFalse(data['normal_app_missing_or_failed_word_environment_qualified'])

    def test_safe_full_validation_before_writes(self):
        docs = fixture()
        out = patch.transform(docs)
        self.assertEqual(set(docs) | {'qualification/evidence/dcm-34-docx-fallback-native-summary.json'}, set(out))
        for key, value in out.items():
            self.assertIsInstance(value, str)
            if key.endswith('.json'):
                json.loads(value)

    def test_summary_source_attribution(self):
        out = patch.transform(fixture())
        data = json.loads(out['qualification/evidence/dcm-34-docx-fallback-native-summary.json'])
        self.assertEqual('bfbb731dc131e2ba691c432555aff2641f44af8a', data['tested_source'])
        self.assertEqual(396, data['native_gates']['baseline']['passed'])
        self.assertEqual(0, data['native_gates']['baseline']['skipped'])
        self.assertEqual(2, data['red']['failed_top_level'])


if __name__ == '__main__':
    unittest.main()
