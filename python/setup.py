import setuptools

with open("README.md", "r") as fh:
    long_description = fh.read()

setuptools.setup(
    name="gi",
    version="0.9.7",
    author="gopy",
    author_email="randy.oreilly@colorado.edu",
    description="Python wrapper around GoGi GUI framework in Go (golang)",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/goki/gi",
    packages=setuptools.find_packages(),
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: BSD License",
        "Operating System :: OS Independent",
    ],
    include_package_data=True,
)
