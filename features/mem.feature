Feature: Memory FS

    Background:
        Given there is a directory "test" in "mem" fs
        And there is a file "test/file1.txt" in "mem" fs
        And there is a file "test/file2.sh" in "mem" fs with content:
        """
        #!/usr/bin/env bash

        echo "hello"
        """

        And I change "test/file2.sh" permission in "mem" fs to 0755

    Scenario: Basic Assertions
        Then there should be a directory "test" in "mem" fs
        And there should be a file "test/file1.txt" in "mem" fs
        And there should be a file "test/file2.sh" in "mem" fs with content:
        """
        #!/usr/bin/env bash

        echo "hello"
        """

        And directory "test" permission in "mem" fs should be 0755
        And file "test/file2.sh" permission in "mem" fs should be 0755

    Scenario: Tree Contains
        Then there should be these files in "mem" fs:
        """
        - test 'perm:"0755"':
            - file1.txt
            - file2.sh 'perm:"0755"'
        """

        And there should be these files in "test/" in "mem" fs:
        """
        - file1.txt
        - file2.sh 'perm:"0755"'
        """

    Scenario: Tree Equal
        Then there should be only these files in "mem" fs:
        """
        - test 'perm:"0755"':
            - file1.txt
            - file2.sh 'perm:"0755"'
        """

        And there should be only these files in "test/" in "mem" fs:
        """
        - file1.txt
        - file2.sh 'perm:"0755"'
        """
