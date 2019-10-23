@errors
Feature: Error paths

   Check how errors are handled by contractapi

    Scenario: User calls unknown function when contract uses unknown transaction handler
        Given I have created and initialised chaincode "SimpleContract"
        When I submit the "FakeFunction" transaction
            | Some | Args |
        Then I should receive an unsuccessful response "Function FakeFunction not found in contract SimpleContract"

    Scenario: User calls unknown function when contract has set an unknown transaction handler
        Given I have created and initialised chaincode "ExtendedSimpleContract"
        When I submit the "FakeFunction" transaction
            | Some | Args |
        Then I should receive an unsuccessful response "Invalid function FakeFunction passed with args [Some, Args]"

    Scenario: Contract function returns error
        Given I have created and initialised chaincode "SimpleContract"
        When I submit the "Read" transaction
            | MISSING_KEY |
        Then I should receive an unsuccessful response "Cannot read key. Key with id MISSING_KEY does not exist"

    Scenario: User sends bad basic data type
        Given I have created and initialised chaincode "ComplexContract"
        When I submit the "NewObject" transaction
            | OBJECT_1 | {"name": "Andy", "contact": "Leave well alone"} | -10 | ["red", "white", "blue"] |
        Then I should receive an unsuccessful response "Error converting parameter param2. Conversion error Cannot convert passed value -10 to uint"

    Scenario: Users sends bad object data type
    Given I have created and initialised chaincode "ComplexContract"
        When I submit the "NewObject" transaction
            | OBJECT_1 | {"firstname": "Andy", "contact": "Leave well alone"} | 1000 | ["red", "white", "blue"] |
        Then I should receive an unsuccessful response "Error validating parameter param1. Value passed for parameter did not match schema:\n1. prop: name is required\n2. prop: Additional property firstname is not allowed"