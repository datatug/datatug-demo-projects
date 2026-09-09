SELECT
    il.InvoiceLineId,
    t.Name AS TrackName,
    il.UnitPrice,
    il.Quantity,
    (il.UnitPrice * il.Quantity) AS LineTotal
FROM InvoiceLine AS il
INNER JOIN Track AS t ON t.TrackId = il.TrackId
WHERE il.InvoiceId = @InvoiceId
ORDER BY il.InvoiceLineId
