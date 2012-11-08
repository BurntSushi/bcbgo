#include <stdio.h>
#include <stdlib.h>

void
MatSet(double **coords, int col, double x, double y, double z)
{
    coords[0][col] = x;
    coords[1][col] = y;
    coords[2][col] = z;
}

double **
MatInit(const int rows, const int cols)
{
    int             i;
    double        **matrix = NULL;
    double         *matspace = NULL;

    matspace = (double *) calloc((rows * cols), sizeof(double));
    if (matspace == NULL) {
        perror("\n ERROR");
        printf("\n ERROR: Failure to allocate matrix space in MatInit(): (%d x %d)\n", rows, cols);
        exit(EXIT_FAILURE);
    }

    /* allocate room for the pointers to the rows */
    matrix = (double **) malloc(rows * sizeof(double *));
    if (matrix == NULL) {
        perror("\n ERROR");
        printf("\n ERROR: Failure to allocate room for row pointers in MatInit(): (%d)\n", rows);
        exit(EXIT_FAILURE);
    }

    /*  now 'point' the pointers */
    for (i = 0; i < rows; i++)
        matrix[i] = matspace + (i * cols);

    return matrix;
}

void
MatDestroy(double **matrix)
{
    if (matrix != NULL) {
        if (matrix[0] != NULL) {
            free(matrix[0]);
            matrix[0] = NULL;
        }
        free(matrix);
    }
}
