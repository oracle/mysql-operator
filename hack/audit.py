# Audit.py
#
# A script to verify the exact SHA of explicit third party dependencies.
#
# Run with: python hack/audit.py && open audit.csv
#
import pytoml as toml
import os

ALIASES = {
    'gopkg.in/ini.v1': 'github.com/go-ini/ini',
}

NORMALIZATIONS = {
    'k8s.io': 'github.com/kubernetes',
    'gopkg.in/yaml.v2': 'github.com/go-yaml/yaml'
}

# Returns a map of dependency to Git SHA
def third_party_constraints(locked):
    with open('Gopkg.toml', 'rb') as f:
        obj = toml.load(f)
        result = {}
        for constraint in obj['constraint']:
            dep = constraint['name']
            if dep in ALIASES:
                dep = ALIASES[dep]
            if dep in locked:
                result[dep] = locked[dep]
            else:
                print('Warning: Could not find revision for %s' % dep)
        return result


def locked_versions():
    with open('Gopkg.lock', 'rb') as f:
      obj = toml.load(f)
      return {project['name']: project['revision'] for project in obj['projects']}


# Returns a dict of dependency name => SHA revision
def generate_dependency_revision_mapping():
    locked = locked_versions()
    return third_party_constraints(locked)


def normalize_dependency_name(dep):
    for k,v in NORMALIZATIONS.iteritems():
        if dep.startswith(k):
            return dep.replace(k,v)
    return dep


def generate_dependency_csv():
    deps = generate_dependency_revision_mapping()
    with open('audit.csv', 'wb') as f:
        f.write('dependency,SHA\n')
        for dep,sha in deps.iteritems():
            dependency_name = normalize_dependency_name(dep)
            f.write('%s,%s\n' % (dependency_name,sha))


if __name__ == '__main__':
    print('Generating CSV of dependencies')
    generate_dependency_csv()
