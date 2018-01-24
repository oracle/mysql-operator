#!/usr/bin/env python
"""This script will gather the exact Git SHA for all explicit third party dependencies"""

import pytoml as toml

ALIASES = {
    'gopkg.in/ini.v1': 'github.com/go-ini/ini',
}

NORMALIZATIONS = {
    'k8s.io': 'github.com/kubernetes',
    'gopkg.in/yaml.v2': 'github.com/go-yaml/yaml'
}

def third_party_constraints(locked):
    """ Generates a map of dep : SHA for third party depedencies """
    with open('Gopkg.toml', 'rb') as fio:
        obj = toml.load(fio)
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
    """ Extracts the dependency name and Git SHA revision for all locked dependencies """
    with open('Gopkg.lock', 'rb') as fio:
        obj = toml.load(fio)
        return {project['name']: project['revision'] for project in obj['projects']}


def generate_revision_mapping():
    """ Returns a dict of dependency name => SHA revision """
    locked = locked_versions()
    return third_party_constraints(locked)


def normalize_dependency_name(dep):
    """ Normalise a dependency name so that we use the Github project name rather than
        something like k8s.io """
    for key, value in NORMALIZATIONS.iteritems():
        if dep.startswith(key):
            return dep.replace(key, value)
    return dep


def short_name(dep):
    """ Returns the short name of a dependency i.e github.com/go-yaml/yaml => yaml """
    return dep.split('/')[-1]


def generate_dependency_csv():
    """ Generates a CSV file """
    deps = generate_revision_mapping()
    with open('audit.csv', 'wb') as fio:
        fio.write('Dependency,Short Name,SHA\n')
        for dep, sha in deps.iteritems():
            dependency_name = normalize_dependency_name(dep)
            dependency_short_name = short_name(dependency_name)
            fio.write('%s,%s,%s\n' % (dependency_name, dependency_short_name, sha))


if __name__ == '__main__':
    print('Generating CSV of dependencies')
    generate_dependency_csv()
