SELECT
    g.Name AS GenreName,
    COUNT(il.InvoiceLineId) AS TracksPurchased,
    SUM(il.UnitPrice * il.Quantity) AS TotalSpent
FROM InvoiceLine AS il
INNER JOIN Invoice AS i ON i.InvoiceId = il.InvoiceId
INNER JOIN Track AS t ON t.TrackId = il.TrackId
INNER JOIN Genre AS g ON g.GenreId = t.GenreId
WHERE i.CustomerId = @CustomerId
GROUP BY g.Name
ORDER BY TotalSpent DESC
