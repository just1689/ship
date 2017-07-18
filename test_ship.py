import os
import sys
import json
import yaml
import pytest
from copy import deepcopy
from io import StringIO
from click.testing import CliRunner
from unittest.mock import Mock
import ship

VALUES_SCHEMA = open('values.schema', 'r').read()

CONFIG = {
    "paths": {
        "terraform": "/bin/echo",
        "kops": "/bin/echo",
        "outputDir": ".",
        "terraformTemplatePath": "networking.tf.template",
        "kopsTemplatePath": "kops.yaml.template",
        "sshPublicKeyPath": "kops.yaml.template"
    },
    "clusterConfig": {
        "FullyQualifiedName": "test.aws.testcompany.tech",
        "ShortName": "test",
        "ConfigBaseURL": "s3://testcompany-kops-config",
        "AWSRegion": "eu-west-2",
        "AWSAZ1": "a",
        "MasterMachineType": "t2.small",
        "MasterMinNumber": 2,
        "MasterMaxNumber": 2,
        "NodeMachineType": "m4.large",
        "NodeMaxNumber": 2,
        "NodeMinNumber": 2,
        "BastionMachineType": "t2.micro",
        "BastionMaxNumber": 1,
        "BastionMinNumber": 1,
    }
}

TERRAFORM_TEMPLATE = {
    "template_vars": [
        "{{FullyQualifiedName}}",
        "{{ShortName}}",
        "{{AWSRegion}}",
        "{{AWSAZ1}}"
    ],
    "otherfield": "othervalue"
}

EXPECTED_TERRAFORM_CONFIG = {
    "template_vars": [
        CONFIG["clusterConfig"]["FullyQualifiedName"],
        CONFIG["clusterConfig"]["ShortName"],
        CONFIG["clusterConfig"]["AWSRegion"],
        CONFIG["clusterConfig"]["AWSAZ1"]
    ],
    "otherfield": "othervalue"
}

KOPS_TEMPLATE = {
    "template_vars": [
        "{{FullyQualifiedName}}",
        "{{ShortName}}",
        "{{ConfigBaseURL}}",
        "{{AWSRegion}}",
        "{{AWSAZ1}}",
        "{{MasterMachineType}}",
        "{{MasterMinNumber}}",
        "{{MasterMaxNumber}}",
        "{{NodeMachineType}}",
        "{{NodeMaxNumber}}",
        "{{NodeMinNumber}}",
        "{{BastionMachineType}}",
        "{{BastionMaxNumber}}",
        "{{BastionMinNumber}}"
    ],
    "otherfield": "othervalue"
}

EXPECTED_KOPS_CONFIG = {
    "template_vars": [
        CONFIG["clusterConfig"]["FullyQualifiedName"],
        CONFIG["clusterConfig"]["ShortName"],
        CONFIG["clusterConfig"]["ConfigBaseURL"],
        CONFIG["clusterConfig"]["AWSRegion"],
        CONFIG["clusterConfig"]["AWSAZ1"],
        CONFIG["clusterConfig"]["MasterMachineType"],
        "{}".format(CONFIG["clusterConfig"]["MasterMinNumber"]),
        "{}".format(CONFIG["clusterConfig"]["MasterMaxNumber"]),
        CONFIG["clusterConfig"]["NodeMachineType"],
        "{}".format(CONFIG["clusterConfig"]["NodeMinNumber"]),
        "{}".format(CONFIG["clusterConfig"]["NodeMaxNumber"]),
        CONFIG["clusterConfig"]["BastionMachineType"],
        "{}".format(CONFIG["clusterConfig"]["BastionMinNumber"]),
        "{}".format(CONFIG["clusterConfig"]["BastionMaxNumber"])
    ],
    "otherfield": "othervalue"
}

TERRAFORM_STATE = {
    "modules": [
        {
            "resources": {
                "aws_vpc." + CONFIG["clusterConfig"]["ShortName"]: {
                    "primary": {
                        "id": "test-vpc-id"
                    }
                },
                "aws_nat_gateway.public-eu-west-2a": {
                    "primary": {
                        "id": "test-nat-gateway-id"
                    }
                },
                "aws_subnet.nodes-eu-west-2a": {
                    "primary": {
                        "id": "test-nodes-subnet-id"
                    }
                },
                "aws_subnet.public-eu-west-2a": {
                    "primary": {
                        "id": "test-public-subnet-id"
                    }
                }
            }
        }
    ]
}

RUNNER = CliRunner()

def test_create_happy_path(capfd):
    expected_output = [
        "apply -state={0}/terraform.tfstate {0}".format(CONFIG['paths']['outputDir']),
        "create -f {}/kops.config --state {}".format(CONFIG['paths']['outputDir'], CONFIG['clusterConfig']['ConfigBaseURL']),
        "create secret --name {} sshpublickey admin -i {} --state {}".format(CONFIG['clusterConfig']['FullyQualifiedName'], CONFIG['paths']['sshPublicKeyPath'], CONFIG['clusterConfig']['ConfigBaseURL']),
        "update cluster --name {} --state {} --yes".format(CONFIG['clusterConfig']['FullyQualifiedName'], CONFIG['clusterConfig']['ConfigBaseURL']),
    ]
    with RUNNER.isolated_filesystem():
        write_configs()

        result = RUNNER.invoke(ship.cli, ['create'])
        out, _ = capfd.readouterr()
        print(result.output)

        assert result.exit_code == 0
        assert_configs_exist()
        assert ordered_output_subset_matches(out, expected_output) == expected_output

def test_update_happy_path(capfd):
    expected_output = [
        "apply -state={0}/terraform.tfstate {0}".format(CONFIG['paths']['outputDir']),
        "replace cluster -f {}/kops.config --state {}".format(CONFIG['paths']['outputDir'], CONFIG['clusterConfig']['ConfigBaseURL']),
        "update cluster --name {} --state {} --yes".format(CONFIG['clusterConfig']['FullyQualifiedName'], CONFIG['clusterConfig']['ConfigBaseURL']),
        "rolling-update cluster --name {} --state {}".format(CONFIG['clusterConfig']['FullyQualifiedName'], CONFIG['clusterConfig']['ConfigBaseURL'])
    ]

    with RUNNER.isolated_filesystem():
        write_configs()

        result = RUNNER.invoke(ship.cli, ['update', '--yes'])
        out, _ = capfd.readouterr()

        assert result.exit_code == 0
        assert_configs_exist()
        assert ordered_output_subset_matches(out, expected_output) == expected_output

def test_destroy_happy_path(capfd):
    expected_output = [
        "delete cluster {} --state {} --yes".format(CONFIG['clusterConfig']['FullyQualifiedName'], CONFIG['clusterConfig']['ConfigBaseURL']),
        "destroy -force {0}".format(CONFIG['paths']['outputDir'])
    ]

    with RUNNER.isolated_filesystem():
        write_configs()
        result = RUNNER.invoke(ship.cli, ['destroy', '--yes'])
        out, _ = capfd.readouterr()

        assert result.exit_code == 0
        assert ordered_output_subset_matches(out, expected_output) == expected_output

def test_create_validates_config():
    verify_validate_called("create")

def test_update_validates_config():
    verify_validate_called("update")

def test_destroy_validates_config():
    verify_validate_called("destroy")

def test_validate_config_checks_missing_values():
    with pytest.raises(Exception):
        ship.validate_config({"unrelated": "field"})

def test_validate_config_checks_invalid_values():
    values_config = deepcopy(CONFIG)
    values_config['clusterConfig']['MasterMaxNumber'] = "3"
    with pytest.raises(Exception):
        ship.validate_config(values_config)

def test_validate_config_checks_invalid_kops_bin():
    values_config = deepcopy(CONFIG)
    values_config['paths']['kops'] = "/mock/please_not_a_real_bin"

    with pytest.raises(IOError):
        ship.validate_config(values_config)

def test_validate_config_checks_invalid_terraform_bin():
    values_config = deepcopy(CONFIG)
    values_config['paths']['terraform'] = "/mock/please_not_a_real_bin"

    with pytest.raises(IOError):
        ship.validate_config(values_config)

def verify_validate_called(method):
    with RUNNER.isolated_filesystem():
        write_configs()
        ship.validate_config = Mock(wraps=ship.validate_config)

        result = RUNNER.invoke(ship.cli, [method])

        ship.validate_config.assert_called_once()

def write_configs(values_override=None):
    with open('values.yaml', 'w') as values_file:
        if values_override:
            values_config = values_override
        else:
            values_config = CONFIG

        yaml.dump(values_config, values_file)

    with open('values.schema', 'w') as values_schema_file:
        values_schema_file.write(VALUES_SCHEMA)

    with open(CONFIG['paths']['terraformTemplatePath'], 'w') as terraform_template_file:
        json.dump(TERRAFORM_TEMPLATE, terraform_template_file)

    with open('terraform.tfstate', 'w') as terraform_state_file:
        json.dump(TERRAFORM_STATE, terraform_state_file)

    with open(CONFIG['paths']['kopsTemplatePath'], 'w') as kops_template_file:
        json.dump(KOPS_TEMPLATE, kops_template_file)

def assert_configs_exist():
    assert os.path.isfile('networking.tf')
    assert os.path.isfile('kops.config')
    with open('networking.tf', 'r') as terraform_config:
        assert json.load(terraform_config) == EXPECTED_TERRAFORM_CONFIG
    with open('kops.config', 'r') as kops_config:
        assert json.load(kops_config) == EXPECTED_KOPS_CONFIG

def ordered_output_subset_matches(output, expected_output):
    output_lines = output.split('\n')
    line_index = 0

    matched_lines = []
    for expected_line in expected_output:
        while output_lines[line_index].strip() != expected_line:
            line_index += 1
            if line_index >= len(output_lines):
                return matched_lines

        matched_lines.append(expected_line)
    return matched_lines
