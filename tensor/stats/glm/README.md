# GLM = general linear model

GLM contains results and parameters for running a [general linear model](https://en.wikipedia.org/wiki/General_linear_model), which is a general form of multivariate linear regression, supporting multiple independent and dependent variables.

Make a `NewGLM` and then do `Run()` on a tensor [IndexView](../table/IndexView) with the relevant data in columns of the table.

# Fitting Methods

## Standard QR Decomposition

The standard algorithm involves eigenvalue computation using [QR Decomposition](https://en.wikipedia.org/wiki/QR_decomposition).  TODO.

## Iterative Batch Mode Least Squares

Batch-mode gradient descent is used and the relevant parameters can be altered from defaults before calling Run as needed.

This mode supports [Ridge](https://en.wikipedia.org/wiki/Ridge_regression) (L2 norm) and [Lasso](https://en.wikipedia.org/wiki/Lasso_(statistics)) (L1 norm) forms of regression, which add different forms of weight decay to the LMS cost function.

