import copy
import hashlib
import importlib.util
import tempfile
import unittest
from pathlib import Path
spec=importlib.util.spec_from_file_location('reviewed',Path(__file__).resolve().parents[1]/'tools/apply_reviewed_patch.py')
patch=importlib.util.module_from_spec(spec);spec.loader.exec_module(patch)

def digest(s): return hashlib.sha256(s.encode()).hexdigest()

class ReviewedPatchTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup)
        self.root=Path(self.temp.name);(self.root/'a').write_text('old\n')
        self.row={'path':'a','before_sha256':digest('old\n'),'replacements':[['old','new']],'after_sha256':digest('new\n')}
        self.spec={'schema':1,'message':'test','edits':[self.row]}
    def test_blank(self):
        with self.assertRaises(ValueError): patch.prepare(self.root,{})
    def test_success_and_no_writes(self):
        self.assertEqual({'a':b'new\n'},patch.prepare(self.root,self.spec))
        self.assertEqual('old\n',(self.root/'a').read_text())
    def test_wrong_before(self):
        self.row['before_sha256']=digest('other')
        with self.assertRaises(ValueError): patch.prepare(self.root,self.spec)
    def test_wrong_after(self):
        self.row['after_sha256']=digest('other')
        with self.assertRaises(ValueError): patch.prepare(self.root,self.spec)
    def test_nonunique(self):
        self.row['replacements']=[['missing','new']]
        with self.assertRaises(ValueError): patch.prepare(self.root,self.spec)
    def test_new_file(self):
        self.row.update(path='b',before_sha256=None,new_content='new\n')
        self.assertEqual({'b':b'new\n'},patch.prepare(self.root,self.spec))
    def test_unsafe_duplicate(self):
        for path in ('../bad','/tmp/bad','.git/config','a\\b'):
            x=copy.deepcopy(self.spec);x['edits'][0]['path']=path
            with self.assertRaises(ValueError): patch.prepare(self.root,x)
        self.spec['edits'].append(copy.deepcopy(self.row))
        with self.assertRaises(ValueError): patch.prepare(self.root,self.spec)
    def test_late_failure_no_writes(self):
        row=copy.deepcopy(self.row);row.update(path='b',before_sha256=digest('missing'))
        self.spec['edits'].append(row)
        with self.assertRaises(ValueError): patch.prepare(self.root,self.spec)
        self.assertEqual('old\n',(self.root/'a').read_text())
if __name__=='__main__': unittest.main()
