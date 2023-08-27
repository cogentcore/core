import setuptools

with open("README.md", "r") as fh:
    long_description = fh.read()

setuptools.setup(
    name="gi",
    version="1.1.0",
    author="gopy",
    author_email="oreilly@ucdavis.edu",
    description="Python wrapper around GoGi GUI framework in Go (golang)",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://goki.dev/gi/v2",
    packages=setuptools.find_packages(),
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: BSD License",
        "Operating System :: OS Independent",
    ],
    include_package_data=True,
)
